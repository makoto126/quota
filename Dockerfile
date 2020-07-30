FROM alpine

ADD out/quota /usr/local/bin
CMD ["quota"]