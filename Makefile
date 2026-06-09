docker-build:
	docker build -t filmmash-go .

docker-run:
	docker rm filmmash-go
	docker run --env-file .env -p 8000:8080 --name filmmash-go -v filmmash_logs:/logs filmmash-go

docker-shell:
	docker exec -it filmmash-go sh

docker-stop:
	docker stop filmmash-go
