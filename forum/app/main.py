from fastapi import FastAPI

from .routes import authors, forum, graph_stats, tags

app = FastAPI()


@app.get("/health")
async def health_check():
    return {"status": "ok"}


app.include_router(forum.router)
app.include_router(authors.router)
app.include_router(tags.router)
app.include_router(graph_stats.router)


@app.on_event("startup")
async def startup_event():
    from .database import init_db

    await init_db()
