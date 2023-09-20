FROM golang:1.21

COPY . /rssnix
RUN cd /rssnix && go install

ENTRYPOINT ["rssnix"]
