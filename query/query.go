package query

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
)

const TableName = "events"

type Event struct {
	ID              string `json:"id"`
	OccurredAt      string `json:"occurred_at"`
	Description     string `json:"description"`
	Title           string `json:"title"`
	ShipmentStepsID string `json:"shipment_steps_id"`
	ExpiresAt       int64  `json:"expires_at"`
	CreatedAt       string `json:"created_at"`
	ServiceStatus   string `json:"service_status"`
}

func Connect(url string) (c *dynamodb.DynamoDB, err error) {
	s, err := session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials("FOO", "BAR", ""),
		Region:      aws.String("DEFAULT_REGION"),
		Endpoint:    aws.String(url),
	})
	if err != nil {
		return
	}

	c = dynamodb.New(s)
	return
}

func Query(client *dynamodb.DynamoDB, date string) ([]Event, error) {
	filter := expression.Name("created_at").Contains(date)
	proj := expression.NamesList(
		expression.Name("id"),
		expression.Name("occurred_at"),
		expression.Name("description"),
		expression.Name("title"),
		expression.Name("shipment_steps_id"),
		expression.Name("expires_at"),
		expression.Name("created_at"),
		expression.Name("service_status"),
	)

	expr, err := expression.NewBuilder().WithFilter(filter).WithProjection(proj).Build()
	if err != nil {
		return nil, err
	}

	input := &dynamodb.ScanInput{
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		FilterExpression:          expr.Filter(),
		ProjectionExpression:      expr.Projection(),
		TableName:                 aws.String(TableName),
	}

	items := []Event{}
	var pageNum int
	if err := client.ScanPages(input,
		func(page *dynamodb.ScanOutput, lastPage bool) bool {
			pageNum++
			fmt.Printf("page %d with %d items\n", pageNum, len(page.Items))
			items = populate(page, items)

			return !lastPage
		},
	); err != nil {
		return nil, err
	}

	return items, nil
}

func populate(items *dynamodb.ScanOutput, e []Event) []Event {
	for _, i := range items.Items {
		item := Event{}

		err := dynamodbattribute.UnmarshalMap(i, &item)
		if err != nil {
			continue
		}

		e = append(e, item)
	}
	return e
}
