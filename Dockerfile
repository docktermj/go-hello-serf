FROM centos:7

ENV REFRESHED_AT 2017-12-19

ARG PROGRAM_NAME="unknown"
ARG BUILD_VERSION=0.0.0
ARG BUILD_ITERATION=0

# --- Install Effing Package Manger (FPM) -------------------------------------

# Install dependencies
RUN yum -y install \
  gcc \
  make \
  rpm-build \
  ruby-devel \
  rubygems \
  which

# Install FPM
RUN gem install --no-ri --no-rdoc fpm

# --- Install Go --------------------------------------------------------------

ENV GO_VERSION=1.9.2

# Install dependencies.
RUN yum -y install \
    git \
    tar \
    wget

# Install "go".
RUN wget https://storage.googleapis.com/golang/go${GO_VERSION}.linux-amd64.tar.gz && \
    tar -C /usr/local/ -xzf go${GO_VERSION}.linux-amd64.tar.gz

# --- Compile go program ------------------------------------------------------

ENV HOME="/root"
ENV GOPATH="${HOME}/gocode"
ENV PATH="${PATH}:/usr/local/go/bin:${GOPATH}/bin"
ENV GO_PACKAGE="github.com/docktermj/${PROGRAM_NAME}"

# Copy local files from the Git repository.
COPY . ${GOPATH}/src/${GO_PACKAGE}

# Install dependencies.
WORKDIR ${GOPATH}/src/${GO_PACKAGE}
RUN go get -u github.com/golang/dep/cmd/dep
RUN dep ensure

# Build go program.
RUN go install \
    -ldflags "-X main.programName=${PROGRAM_NAME} -X main.buildVersion=${BUILD_VERSION} -X main.buildIteration=${BUILD_ITERATION}" \
    ${GO_PACKAGE}

# Copy binary to output.
RUN mkdir -p /output/bin && \
    cp /root/gocode/bin/${PROGRAM_NAME} /output/bin

# --- Package as RPM and DEB --------------------------------------------------

WORKDIR /output

# RPM package.
RUN fpm \
  --input-type dir \
  --output-type rpm \
  --name ${PROGRAM_NAME} \
  --version ${BUILD_VERSION} \
  --iteration ${BUILD_ITERATION} \
  /root/gocode/bin/=/usr/bin

# DEB package.
RUN fpm \
  --input-type dir \
  --output-type deb \
  --name ${PROGRAM_NAME} \
  --version ${BUILD_VERSION} \
  --iteration ${BUILD_ITERATION} \
  /root/gocode/bin/=/usr/bin

# --- Epilog ------------------------------------------------------------------

CMD ["/bin/bash"]
