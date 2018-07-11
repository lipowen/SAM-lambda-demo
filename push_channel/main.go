package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"

	_ "github.com/go-sql-driver/mysql"
)

const awsRegion = "ap-southeast-1"

type SubscribedTopics struct {
	Logout           bool
	SubscriptionArns []string `json:"subscription_arns"`
}

type SnsInfo struct {
	TopicArn string `json:"topic_arn"`
	Protocol string
	EndPoint string
}

func Subscribe(snsInfo SnsInfo) (string, error) {
	sess, err := session.NewSession(&aws.Config{Region: aws.String(awsRegion)})

	input := &sns.SubscribeInput{
		Endpoint: &snsInfo.EndPoint,
		Protocol: &snsInfo.Protocol,
		TopicArn: &snsInfo.TopicArn,
	}

	if err != nil {
		return "", errors.New("Unable to initiate session")
	}

	svc := sns.New(sess)
	out, err := svc.Subscribe(input)

	if err != nil {
		fmt.Println("Unable to Subscribe")
		return "", errors.New("Unable to Subscribe")
	}

	return *out.SubscriptionArn, nil
}

func LoadSubscriptionArns(AmazonSnsInfoID int) (topics string) {
	var queryResult string

	db, _ := sql.Open("mysql", os.Getenv("DB_URL"))
	row := db.QueryRow("SELECT subscribed_topics FROM amazon_sns_infos WHERE id = ?", AmazonSnsInfoID)
	row.Scan(&queryResult)

	return queryResult
}

// SaveResponse is unexported type
func SaveResponse(AmazonSnsInfoID int, topic string) {
	var parseResult SubscribedTopics

	queryResult := LoadSubscriptionArns(AmazonSnsInfoID)

	json.Unmarshal([]byte(queryResult), &parseResult)
	parseResult.SubscriptionArns = append(parseResult.SubscriptionArns, topic)

	db, _ := sql.Open("mysql", os.Getenv("DB_URL"))
	query, _ := db.Prepare("UPDATE amazon_sns_infos set subscribed_topics=? where id=?")
	newSubscribedTopics, _ := json.Marshal(parseResult)

	query.Exec(newSubscribedTopics, AmazonSnsInfoID)
	query.Close()
}

func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var snsInfo SnsInfo
	amazonSnsInfoID, _ := strconv.Atoi(request.PathParameters["id"])

	json.Unmarshal([]byte(request.Body), &snsInfo)

	topic, _ := Subscribe(snsInfo)

	SaveResponse(amazonSnsInfoID, topic)

	return events.APIGatewayProxyResponse{
		Body:       "",
		StatusCode: 200,
	}, nil
}

func main() {
	lambda.Start(Handler)
}
