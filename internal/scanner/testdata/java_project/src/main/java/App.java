import software.amazon.awssdk.services.s3.S3Client;

public class App {
    public static void main(String[] args) {
        S3Client s3 = S3Client.create();
    }
}
