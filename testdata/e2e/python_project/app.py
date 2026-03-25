import boto3

def upload_file(bucket, key, body):
    s3 = boto3.client('s3')
    s3.put_object(Bucket=bucket, Key=key, Body=body)

def send_message(queue_url, message):
    sqs = boto3.client('sqs')
    sqs.send_message(QueueUrl=queue_url, MessageBody=message)

def publish_notification(topic_arn, message):
    sns = boto3.client('sns')
    sns.publish(TopicArn=topic_arn, Message=message)

def get_secret(secret_name):
    sm = boto3.client('secretsmanager')
    return sm.get_secret_value(SecretId=secret_name)