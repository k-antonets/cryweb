FROM lab7arriam/cry_processor:latest

RUN pip install celery

ADD ./tasks/ /app/

WORKDIR /app

ENTRYPOINT celery -A tasks worker -l info