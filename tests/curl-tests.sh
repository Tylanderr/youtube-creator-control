curl -X POST http://localhost:8080/post -d "$(cat ./json/request.json)" -H "Content-Type: application/json"

curl -X POST http://localhost:8080/postMedia -F "file=@tests/media/4Vkj2WvisFNsbs0tS6fjRqug2pgr6Bf0WFQO5x3tJkk.jpg" -H "Content-Type: multipart/form-data"
