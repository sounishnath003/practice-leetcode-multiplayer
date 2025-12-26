
.PHONY:
run:
	FLASK_APP=main:app uv run flask run --port 8080	

.PHONY: deploy
deploy:
	gcloud run deploy code-execution-engine --source . --region asia-south1 --allow-unauthenticated