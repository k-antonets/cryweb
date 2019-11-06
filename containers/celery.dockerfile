FROM lab7arriam/cry_processor:latest

RUN apt update -y && apt install -y zip unzip && pip install -U "celery[redis]"

ADD ./tasks/ /app/

WORKDIR /app

ENTRYPOINT celery -A tasks worker -l info -Q cry_py