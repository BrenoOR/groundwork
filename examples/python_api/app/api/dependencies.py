from functools import lru_cache

from fastapi import Depends

from app.repositories.storage_repository import S3StorageRepository
from app.repositories.user_repository import InMemoryUserRepository
from app.services.file_access_service import FileAccessService


@lru_cache
def get_user_repository() -> InMemoryUserRepository:
    return InMemoryUserRepository()


@lru_cache
def get_storage_repository() -> S3StorageRepository:
    return S3StorageRepository()


def get_file_access_service(
    user_repo: InMemoryUserRepository = Depends(get_user_repository),
    storage_repo: S3StorageRepository = Depends(get_storage_repository),
) -> FileAccessService:
    return FileAccessService(user_repo=user_repo, storage_repo=storage_repo)
