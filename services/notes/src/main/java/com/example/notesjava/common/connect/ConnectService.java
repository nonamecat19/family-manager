package com.example.notesjava.common.connect;

import com.google.protobuf.Message;

import java.util.LinkedHashMap;
import java.util.Map;

public abstract class ConnectService {

    private final Map<String, Procedure<?, ?>> procedures = new LinkedHashMap<>();

    public abstract String serviceName();

    protected <I extends Message, O extends Message> void register(
            String method, I prototype, UnaryHandler<I, O> handler) {
        procedures.put(method, new Procedure<>(prototype, handler));
    }

    public Procedure<?, ?> procedure(String method) {
        Procedure<?, ?> procedure = procedures.get(method);
        if (procedure == null) {
            throw ConnectException.unimplemented(serviceName() + "/" + method);
        }
        return procedure;
    }

    public record Procedure<I extends Message, O extends Message>(I prototype, UnaryHandler<I, O> handler) {

        @SuppressWarnings("unchecked")
        public Message invoke(Message request) {
            return handler.handle((I) request);
        }
    }
}
