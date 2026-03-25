from app.domain.models import Team, User


class InMemoryUserRepository:
    """
    Single Responsibility: manages user and team data only.
    In production this would be backed by a database.
    """

    def __init__(self) -> None:
        self._teams: dict[str, Team] = {
            "team-alpha": Team(id="team-alpha", name="Alpha"),
            "team-beta": Team(id="team-beta", name="Beta"),
        }
        self._users: dict[str, User] = {
            "user-1": User(
                id="user-1",
                name="Alice",
                email="alice@example.com",
                team_ids=["team-alpha"],
            ),
            "user-2": User(
                id="user-2",
                name="Bob",
                email="bob@example.com",
                team_ids=["team-alpha", "team-beta"],
            ),
            "user-3": User(
                id="user-3",
                name="Carol",
                email="carol@example.com",
                team_ids=["team-beta"],
            ),
        }

    def get_by_id(self, user_id: str) -> User | None:
        return self._users.get(user_id)

    def get_team_by_id(self, team_id: str) -> Team | None:
        return self._teams.get(team_id)
