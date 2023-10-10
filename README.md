# CheckInBot
Automated talent check-in reporting for employees and managers.  
Let's make each check-in more personal and meaningful with real-time data collated alongside professional goals and employee feedback. 
Plus, it's a great conversation starter for any 1:1 meeting.  Availabile on-demand to your email inbox.

This project was created as practice in the go programming language. 

## Initial Deployment steps:

$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build -o main
zip deployment.zip main
aws iam create-role --role-name CheckInBotExecutionRole --assume-role-policy-document file://trust-policy.json
aws iam attach-role-policy --role-name CheckInBotExecutionRole --policy-arn arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole

aws iam create-policy --policy-name SES-SendEmail-Policy --policy-document file://ses-sendemail-policy.json

aws iam attach-role-policy --role-name CheckInBotExecutionRole --policy-arn <YOUR_SES_POLICY_ARN>

aws iam create-policy --policy-name SecretsManagerCheckInBotConfigAccess --policy-document file://secrets_policy.json

aws iam attach-role-policy --role-name CheckInBotExecutionRole --policy-arn <YOUR_SECRETS_POLICY_ARN>

aws lambda create-function --function-name CheckInBotFunction --runtime go1.x --role arn:aws:iam::<YOUR_ACCOUNT_ID>:role/CheckInBotExecutionRole --handler main --zip-file fileb://deployment.zip --region <YOUR_REGION>

aws lambda update-function-configuration --function-name CheckInBotFunction --timeout 30 --region <YOUR_REGION>

aws apigateway create-rest-api --name CheckInBotAPI --region <YOUR_REGION>

aws apigateway get-resources --rest-api-id <YOUR_API_ID> --region <YOUR_REGION>

aws apigateway put-method --rest-api-id <YOUR_API_ID> --resource-id <YOUR_RESOURCE_ID> --http-method GET --authorization-type NONE --region <YOUR_REGION>

aws apigateway put-method --rest-api-id <YOUR_API_ID> --resource-id <YOUR_RESOURCE_ID> --http-method OPTIONS --authorization-type NONE --region <YOUR_REGION>

aws apigateway put-integration --rest-api-id <YOUR_API_ID> --resource-id <YOUR_RESOURCE_ID> --http-method OPTIONS --type MOCK --request-templates file://request_templates.json --region <YOUR_REGION>

aws apigateway put-method-response --rest-api-id <YOUR_API_ID> --resource-id <YOUR_RESOURCE_ID> --http-method OPTIONS --status 200 --response-models file://response_models.json --response-parameters file://response_parameters.json --region <YOUR_REGION>

aws apigateway put-integration-response --rest-api-id <YOUR_API_ID> --resource-id <YOUR_RESOURCE_ID> --http-method OPTIONS --status 200 --response-templates file://response_templates.json --response-parameters file://response_parameters_integration.json --region <YOUR_REGION>

aws apigateway put-integration --rest-api-id <YOUR_API_ID> --resource-id <YOUR_RESOURCE_ID> --http-method GET --type AWS_PROXY --integration-http-method POST --uri arn:aws:apigateway:<YOUR_REGION>:lambda:path/2015-03-31/functions/arn:aws:lambda:<YOUR_REGION>:<YOUR_ACCOUNT_ID>:function:CheckInBotFunction/invocations --region <YOUR_REGION>

aws apigateway create-deployment --rest-api-id <YOUR_API_ID> --stage-name bot --stage-description 'Practice Stage' --description 'First Deployment' --region <YOUR_REGION>

aws lambda add-permission --function-name CheckInBotFunction --statement-id apigateway-bot-1 --action lambda:InvokeFunction --principal apigateway.amazonaws.com --source-arn "arn:aws:execute-api:<YOUR_REGION>:<YOUR_ACCOUNT_ID>:<YOUR_API_ID>/prod/GET/" --region <YOUR_REGION>

aws secretsmanager create-secret --name CheckInBotConfig --description "Bot config params" --secret-string file://config.json --region <YOUR_REGION>


## Usage: 

https://<YOUR_API_ID>.execute-api.<YOUR_REGION>.amazonaws.com/bot?initials=<Manager_Initials>


## Lambda update:

$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build -o main
zip deployment.zip main
aws lambda update-function-code --function-name CheckInBotFunction --zip-file fileb://deployment.zip --region <YOUR_REGION>


## API update:

Make changes in AWS and then re-deploy the API:
aws apigateway create-deployment --rest-api-id YOUR_REST_API_ID --stage-name bot --region <YOUR_REGION>


## Cleanup:

aws lambda delete-function --function-name CheckInBotFunction
aws iam delete-role --role-name CheckInBotExecutionRole
aws apigateway delete-rest-api --rest-api-id <YOUR_API_ID>
