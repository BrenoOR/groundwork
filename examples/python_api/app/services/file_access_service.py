from app.domain.interfaces import IStorageRepository, IUserRepository
from app.domain.models import FileEntry


class FileAccessService:
    """
    Single Responsibility: enforces file-access rules for a user.
    Dependency Inversion: depends on IUserRepository and IStorageRepository,
    not on concrete implementations.
    """

    def __init__(
        self,
        user_repo: IUserRepository,
        storage_repo: IStorageRepository,
    ) -> None:
        self._users = user_repo
        self._storage = storage_repo

    def list_accessible_files(self, user_id: str) -> list[FileEntry]:
        user = self._users.get_by_id(user_id)
        if user is None:
            raise ValueError(f"user {user_id!r} not found")

        files: list[FileEntry] = []

        # Personal folder.
        files.extend(self._storage.list_objects(user.s3_prefix))

        # One shared folder per team the user belongs to.
        for team_id in user.team_ids:
            team = self._users.get_team_by_id(team_id)
            if team is not None:
                files.extend(self._storage.list_objects(team.s3_prefix))

        return files

    def get_download_url(self, user_id: str, key: str) -> str:
        user = self._users.get_by_id(user_id)
        if user is None:
            raise ValueError(f"user {user_id!r} not found")

        if not self._is_accessible(user_id, key):
            raise PermissionError(
                f"user {user_id!r} does not have access to {key!r}"
            )

        return self._storage.presigned_url(key)

    # --- private ---

    def _is_accessible(self, user_id: str, key: str) -> bool:
        """Returns True if the key falls under a prefix the user can access."""
        user = self._users.get_by_id(user_id)
        if user is None:
            return False

        allowed_prefixes = [user.s3_prefix]
        for team_id in user.team_ids:
            team = self._users.get_team_by_id(team_id)
            if team is not None:
                allowed_prefixes.append(team.s3_prefix)

        return any(key.startswith(prefix) for prefix in allowed_prefixes)
