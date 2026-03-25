from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    aws_region: str = "us-east-1"
    s3_bucket: str = "my-company-files"

    model_config = {"env_file": ".env", "env_file_encoding": "utf-8"}


settings = Settings()
