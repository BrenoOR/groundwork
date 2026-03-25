from fastapi import APIRouter, Depends, HTTPException, status
from pydantic import BaseModel

from app.api.dependencies import get_user_repository
from app.repositories.user_repository import InMemoryUserRepository

router = APIRouter(prefix="/users", tags=["users"])


class UserResponse(BaseModel):
    id: str
    name: str
    email: str
    team_ids: list[str]


@router.get("/{user_id}", response_model=UserResponse)
def get_user(
    user_id: str,
    repo: InMemoryUserRepository = Depends(get_user_repository),
) -> UserResponse:
    user = repo.get_by_id(user_id)
    if user is None:
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail="user not found")

    return UserResponse(
        id=user.id,
        name=user.name,
        email=user.email,
        team_ids=user.team_ids,
    )
