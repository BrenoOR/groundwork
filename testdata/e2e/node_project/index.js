const { S3Client, PutObjectCommand } = require('@aws-sdk/client-s3');
const { DynamoDBClient, PutItemCommand } = require('@aws-sdk/client-dynamodb');
const { SNSClient, PublishCommand } = require('@aws-sdk/client-sns');
const { SQSClient, SendMessageCommand } = require('@aws-sdk/client-sqs');
const { LambdaClient, InvokeCommand } = require('@aws-sdk/client-lambda');

const region = process.env.AWS_REGION || 'us-east-1';

const s3 = new S3Client({ region });
const dynamo = new DynamoDBClient({ region });
const sns = new SNSClient({ region });
const sqs = new SQSClient({ region });
const lambda = new LambdaClient({ region });

module.exports = { s3, dynamo, sns, sqs, lambda };