FROM golang:1.4-onbuild

EXPOSE 8080
CMD ["app", "-host", "0.0.0.0", "-port", "8080"]
