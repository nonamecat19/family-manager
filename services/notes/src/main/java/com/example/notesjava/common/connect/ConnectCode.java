package com.example.notesjava.common.connect;

import org.springframework.http.HttpStatus;

public enum ConnectCode {
    INVALID_ARGUMENT("invalid_argument", HttpStatus.BAD_REQUEST),
    UNAUTHENTICATED("unauthenticated", HttpStatus.UNAUTHORIZED),
    PERMISSION_DENIED("permission_denied", HttpStatus.FORBIDDEN),
    NOT_FOUND("not_found", HttpStatus.NOT_FOUND),
    FAILED_PRECONDITION("failed_precondition", HttpStatus.PRECONDITION_FAILED),
    UNIMPLEMENTED("unimplemented", HttpStatus.NOT_IMPLEMENTED),
    UNAVAILABLE("unavailable", HttpStatus.SERVICE_UNAVAILABLE),
    INTERNAL("internal", HttpStatus.INTERNAL_SERVER_ERROR);

    private final String wireName;
    private final HttpStatus status;

    ConnectCode(String wireName, HttpStatus status) {
        this.wireName = wireName;
        this.status = status;
    }

    public String wireName() {
        return wireName;
    }

    public HttpStatus status() {
        return status;
    }
}
