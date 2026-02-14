import os

from dotenv import load_dotenv

load_dotenv()

MONGODB_URL = os.getenv(
    "MONGODB_URL", "mongodb://admin:admin@mongodb:27017/forumdb?authSource=admin"
)
DB_NAME = os.getenv("DB_NAME", "forumdb")
