import boto3
from botocore.exceptions import ClientError

from app.core.config import settings
from app.domain.models import FileEntry


class S3StorageRepository:
    """
    Single Responsibility: all S3 interaction lives here.
    Open/Closed: new storage backends (GCS, Azure Blob) implement IStorageRepository
    without touching this class or the services.
    """

    def __init__(self) -> None:
        self._client = boto3.client("s3", region_name=settings.aws_region)
        self._bucket = settings.s3_bucket

    def list_objects(self, prefix: str) -> list[FileEntry]:
        entries: list[FileEntry] = []
        paginator = self._client.get_paginator("list_objects_v2")

        for page in paginator.paginate(Bucket=self._bucket, Prefix=prefix):
            for obj in page.get("Contents", []):
                key: str = obj["Key"]
                # Skip virtual folder markers.
                if key.endswith("/"):
                    continue
                entries.append(
                    FileEntry(
                        key=key,
                        size=obj["Size"],
                        last_modified=obj["LastModified"].isoformat(),
                        owner=_owner_from_key(key),
                    )
                )

        return entries

    def presigned_url(self, key: str, expires_in: int = 3600) -> str:
        try:
            return self._client.generate_presigned_url(
                "get_object",
                Params={"Bucket": self._bucket, "Key": key},
                ExpiresIn=expires_in,
            )
        except ClientError as exc:
            raise RuntimeError(f"could not generate URL for {key!r}") from exc


def _owner_from_key(key: str) -> str:
    """Derives a human-readable owner label from the S3 key prefix."""
    parts = key.split("/")
    if len(parts) >= 2:
        return parts[1]  # user-id or team-id
    return "unknown"
