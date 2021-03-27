FROM alpine

WORKDIR "/app/cassem"

COPY "./cassemd" .
COPY "./cassemctl" .