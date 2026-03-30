import software.amazon.awssdk.services.s3.S3Client;
import software.amazon.awssdk.services.dynamodb.DynamoDbClient;
import com.amazonaws.services.sqs.AmazonSQS;
import com.amazonaws.services.secretsmanager.AWSSecretsManager;

public class App {
    public static void main(String[] args) {
        S3Client s3 = S3Client.create();
        DynamoDbClient dynamo = DynamoDbClient.create();
    }
}
