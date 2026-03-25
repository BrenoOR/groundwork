from typing import Protocol

from app.domain.models import FileEntry, Team, User


# --- Repository interfaces (Dependency Inversion) ---

class IUserRepository(Protocol):
    def get_by_id(self, user_id: str) -> User | None: ...
    def get_team_by_id(self, team_id: str) -> Team | None: ...


class IStorageRepository(Protocol):
    """Interface Segregation: only the operations the services actually need."""

    def list_objects(self, prefix: str) -> list[FileEntry]: ...
    def presigned_url(self, key: str, expires_in: int = 3600) -> str: ...


# --- Service interfaces ---

class IFileAccessService(Protocol):
    def list_accessible_files(self, user_id: str) -> list[FileEntry]: ...
    def get_download_url(self, user_id: str, key: str) -> str: ...
