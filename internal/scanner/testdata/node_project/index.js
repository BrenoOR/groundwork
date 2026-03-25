const { S3Client } = require('@aws-sdk/client-s3');
const { DynamoDBClient } = require('@aws-sdk/client-dynamodb');

const s3 = new S3Client({ region: 'us-east-1' });
const dynamo = new DynamoDBClient({ region: 'us-east-1' });