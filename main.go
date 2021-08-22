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
	getParamPerson := &dynamodb.QueryInput{
		TableName: aws.String("userActivities"),
		ExpressionAttributeNames: map[string]*string{
			"#ID": aws.String("userID"), // alias付けれたりする
		//	"#Status":aws.String("status"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":userID": {
				S: aws.String(requestItem.UserID),
			},
			/*
			":status":{
				N: aws.String("1"),
			},

			 */
		},
		KeyConditionExpression: aws.String("#ID = :userID"), // 検索条件
		ScanIndexForward:       aws.Bool(false),             // ソートキーのソート順（指定しないと昇順）
		Limit:                  aws.Int64(50),                // 修正します
	}
	// 検索
	getData, err := svc.Query(getParamPerson)
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
