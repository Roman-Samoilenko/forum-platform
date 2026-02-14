from motor.motor_asyncio import AsyncIOMotorClient
from pymongo import TEXT

from .config import DB_NAME, MONGODB_URL

client = AsyncIOMotorClient(MONGODB_URL)
db = client[DB_NAME]

theme_collection = db.theme
users_collection = db.users
tags_collection = db.tags


async def init_db():
    await theme_collection.create_index(
        [("title", TEXT), ("author", TEXT), ("tags", TEXT), ("links", TEXT)],
        name="theme_search_index",
    )

    await users_collection.create_index([("username", TEXT)], name="user_search_index")

    await tags_collection.create_index([("name", TEXT)], name="tags_search_index")
