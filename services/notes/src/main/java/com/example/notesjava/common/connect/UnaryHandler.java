package com.example.notesjava.common.connect;

import com.google.protobuf.Message;

@FunctionalInterface
public interface UnaryHandler<I extends Message, O extends Message> {

    O handle(I request);
}
