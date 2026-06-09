docker-build:
	docker build -t my-app .

docker-run:
	docker rm my-app
	docker run --env-file .env -p 8000:8080 --name my-app -v ./logs:/logs my-app

docker-shell:
	docker exec -it my-app sh

docker-stop:
	docker stop my-app
