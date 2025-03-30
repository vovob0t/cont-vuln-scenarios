from flask import Flask
import os

app = Flask(__name__)

@app.route("/")
def index():
    secret = os.environ.get("MY_SECRET", "Not Set")
    return f"Hello, World! Secret is: {secret}"

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=5000)

