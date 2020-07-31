FROM makoto126/xfsquota

ADD out/quota /usr/local/bin
CMD ["quota"]