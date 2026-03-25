from fastapi import APIRouter, Depends, HTTPException, status
from pydantic import BaseModel

from app.api.dependencies import get_file_access_service
from app.services.file_access_service import FileAccessService

router = APIRouter(prefix="/users/{user_id}/files", tags=["files"])


class FileEntryResponse(BaseModel):
    key: str
    size: int
    last_modified: str
    owner: str


class DownloadUrlResponse(BaseModel):
    url: str
    expires_in: int = 3600


@router.get("", response_model=list[FileEntryResponse])
def list_files(
    user_id: str,
    service: FileAccessService = Depends(get_file_access_service),
) -> list[FileEntryResponse]:
    """List all files accessible by the user (personal + team shared folders)."""
    try:
        entries = service.list_accessible_files(user_id)
    except ValueError as exc:
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail=str(exc))

    return [
        FileEntryResponse(
            key=e.key,
            size=e.size,
            last_modified=e.last_modified,
            owner=e.owner,
        )
        for e in entries
    ]


@router.get("/{file_key:path}/download", response_model=DownloadUrlResponse)
def get_download_url(
    user_id: str,
    file_key: str,
    service: FileAccessService = Depends(get_file_access_service),
) -> DownloadUrlResponse:
    """Generate a presigned S3 URL for a file the user has access to."""
    try:
        url = service.get_download_url(user_id, file_key)
    except ValueError as exc:
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail=str(exc))
    except PermissionError as exc:
        raise HTTPException(status_code=status.HTTP_403_FORBIDDEN, detail=str(exc))

    return DownloadUrlResponse(url=url)
