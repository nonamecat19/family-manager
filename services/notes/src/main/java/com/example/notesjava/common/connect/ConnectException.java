package com.example.notesjava.common.connect;

public class ConnectException extends RuntimeException {

    private final ConnectCode code;

    public ConnectException(ConnectCode code, String message) {
        super(message);
        this.code = code;
    }

    public ConnectCode code() {
        return code;
    }

    public static ConnectException invalidArgument(String message) {
        return new ConnectException(ConnectCode.INVALID_ARGUMENT, message);
    }

    public static ConnectException notFound(String message) {
        return new ConnectException(ConnectCode.NOT_FOUND, message);
    }

    public static ConnectException permissionDenied(String message) {
        return new ConnectException(ConnectCode.PERMISSION_DENIED, message);
    }

    public static ConnectException unavailable(String message) {
        return new ConnectException(ConnectCode.UNAVAILABLE, message);
    }

    public static ConnectException unimplemented(String method) {
        return new ConnectException(ConnectCode.UNIMPLEMENTED,
                method + " is not implemented by this service yet");
    }
}
