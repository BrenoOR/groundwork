from dataclasses import dataclass, field


@dataclass(frozen=True)
class Team:
    id: str
    name: str

    @property
    def s3_prefix(self) -> str:
        return f"teams/{self.id}/"


@dataclass(frozen=True)
class User:
    id: str
    name: str
    email: str
    team_ids: list[str] = field(default_factory=list)

    @property
    def s3_prefix(self) -> str:
        return f"users/{self.id}/"


@dataclass(frozen=True)
class FileEntry:
    key: str
    size: int
    last_modified: str
    owner: str  # "personal" | team_id
