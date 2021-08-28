package main

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"log"
	"strconv"
)

type Item struct {
	UserID    string `dynamodbav:"userID" json:"userID"`
	Timestamp int64    `dynamodbav:"timestamp" json:"timestamp"`
	Status    int    `dynamodbav:"status" json:"status"`
}

type Request struct {
	UserID    string `dynamodbav:"userID" json:"userID"`
	Timestamp int64    `dynamodbav:"timestamp" json:"timestamp"`
	Status    int    `dynamodbav:"status" json:"status"`
}

type Response struct {
	WorkingTime int64 `json:"workingTime"`
}

func WorkingTime(startTime, endTime int64) int64 {
	workingTime := endTime - startTime

	return workingTime
}

func Put(svc *dynamodb.DynamoDB, insert Item) error {
	var err error
	//create put processing
	insertData, err := dynamodbattribute.MarshalMap(insert)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	fmt.Println(insertData)
	putParams := &dynamodb.PutItemInput{
		TableName: aws.String("userActivities"),
		Item:      insertData,
	}
	//Execute.
	_, err = svc.PutItem(putParams)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	return err
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// DB接続
	svc := dynamodb.New(session.New(), aws.NewConfig().WithRegion("ap-northeast-1"))

	log.Println(request.Body)
	requestItem := Request{}
	if err := json.Unmarshal(([]byte)(request.Body), &requestItem); err != nil {
		return events.APIGatewayProxyResponse{
			Body:       err.Error(),
			StatusCode: 500,
		}, err
	}
//検索条件にtimestampを入れりと怒られるので外法を使います
	input := &dynamodb.QueryInput{
		TableName: aws.String("userActivities"),
		ExpressionAttributeNames: map[string]*string{
			"#userID":   aws.String("userID"), // alias付けれたりする
			"#timestamp": aws.String("timestamp"),   // 予約語はそのままだと怒られるので置換する
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":userID": { // :を付けるのがセオリーのようです
				S: aws.String(requestItem.UserID),
			},
			":timestamp": { // :を付けるのがセオリーのようです
				N: aws.String(strconv.Itoa(int(requestItem.Timestamp)-86400)),
			},
		},
		KeyConditionExpression: aws.String("#userID = :userID AND #timestamp > :timestamp"),         // 検索条件
		//ProjectionExpression:   aws.String("#userID, #timestamp, #Name"), // 取得カラム
		ScanIndexForward:       aws.Bool(false),                 // ソートキーのソート順（指定しないと昇順）
	}

	getData, err := svc.Query(input)
	if err != nil {
		fmt.Println("[Query Error]", err)
		return events.APIGatewayProxyResponse{
				Body:       err.Error(),
				StatusCode: 500,
			},
			err
	}

	// 結果を構造体にパース
	items := make([]*Item, 0)

	err = dynamodbattribute.UnmarshalListOfMaps(getData.Items, &items)
	if err != nil {
		return events.APIGatewayProxyResponse{
			Body:       err.Error(),
			StatusCode: 500,
		}, err
	}

	/*
	userActivity := []Item{}
	err = dynamodbattribute.UnmarshalListOfMaps(getData.Items, &userActivity)
	if err != nil {
		fmt.Println(err.Error())
	}


	 */
	insertData := Item{
		UserID:    requestItem.UserID,
		Timestamp: requestItem.Timestamp,
		Status:    4,
	}

	err = Put(svc, insertData)
	if err != nil {
		log.Fatalf("Got error unmarshalling: %s", err)
		return events.APIGatewayProxyResponse{
				Body:       err.Error(),
				StatusCode: 500,
			},
			err
	}

	var workingTime int64

	for _,v := range items{
		if(v.Status == 1 ){
			workingTime = WorkingTime(v.Timestamp, requestItem.Timestamp)
			break;
		}
	}

	//workingTime := WorkingTime(userActivity[0].Timestamp, requestItem.Timestamp)

	responseData := Response{
		WorkingTime: workingTime,
	}

	responseDataJson, _ := json.Marshal(responseData)

	return events.APIGatewayProxyResponse{
		Headers: map[string]string{
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Headers": "origin,Accept,Authorization,Content-Type",
			"Content-Type":                 "application/json",
		},
		Body:       string(responseDataJson),
		StatusCode: 200,
	}, nil

}
func main() {
	lambda.Start(handler)
}
