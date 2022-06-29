package query_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	"github.com/vitorhrmiranda/dynamo/query"

	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	_ "embed"
)

//go:embed db.json
var raw []byte

func Setup(t *testing.T) string {
	t.Helper()

	if b := os.Getenv("URL"); len(b) != 0 {
		return b
	}

	compose := testcontainers.NewLocalDockerCompose(
		[]string{"../docker-compose.yml"},
		strings.ToLower(uuid.New().String()),
	)

	container := compose.
		WithCommand([]string{"up", "-d", "dynamo"}).
		WaitForService("dynamo", wait.NewHostPortStrategy(nat.Port("8000")))

	_ = container.Invoke()

	t.Cleanup(func() {
		compose.Down()
	})

	return "http://0.0.0.0:8000"
}

func CreateTable(t *testing.T, client *dynamodb.DynamoDB) {
	t.Helper()

	t.Log("creating table ...")
	defer t.Log("creating table ... done")

	_, _ = client.DeleteTable(&dynamodb.DeleteTableInput{TableName: aws.String(query.TableName)})

	_, err := client.CreateTable(&dynamodb.CreateTableInput{
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String("id"),
				AttributeType: aws.String("S"),
			},
			{
				AttributeName: aws.String("occurred_at"),
				AttributeType: aws.String("S"),
			},
		},
		KeySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String("id"),
				KeyType:       aws.String("HASH"),
			},
			{
				AttributeName: aws.String("occurred_at"),
				KeyType:       aws.String("RANGE"),
			},
		},
		ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(10),
			WriteCapacityUnits: aws.Int64(10),
		},
		TableName: aws.String(query.TableName),
	})

	assert.NoError(t, err)
}

func Seeds(t *testing.T, client *dynamodb.DynamoDB) {
	t.Helper()

	t.Log("creating seeds ...")
	defer t.Log("creating seeds ... done")

	var items []query.Event
	err := json.Unmarshal(raw, &items)
	assert.NoError(t, err)

	for _, item := range items {
		av, err := dynamodbattribute.MarshalMap(item)
		assert.NoError(t, err)

		input := &dynamodb.PutItemInput{
			Item:      av,
			TableName: aws.String(query.TableName),
		}

		_, err = client.PutItem(input)
		assert.NoError(t, err)
	}
}

func TestQuery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	url := Setup(t)
	client, err := query.Connect(url)
	assert.NoError(t, err)

	CreateTable(t, client)
	Seeds(t, client)

	tests := []struct {
		day  string
		want int
	}{
		{"01/06/2022", 2},
		{"02/06/2022", 3},
		{"03/06/2022", 4},
		{"04/06/2022", 1},
	}

	for _, tt := range tests {
		t.Run("when day is "+tt.day, func(t *testing.T) {
			events, err := query.Query(client, tt.day)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, len(events))
		})
	}
}
