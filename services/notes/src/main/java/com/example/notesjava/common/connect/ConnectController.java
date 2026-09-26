package com.example.notesjava.common.connect;

import com.example.notesjava.common.error.InvalidRequestException;
import com.example.notesjava.common.error.MissingFamilyException;
import com.example.notesjava.common.error.ResourceNotFoundException;
import com.google.protobuf.InvalidProtocolBufferException;
import com.google.protobuf.Message;
import com.google.protobuf.util.JsonFormat;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.http.HttpHeaders;
import org.springframework.http.MediaType;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestHeader;
import org.springframework.web.bind.annotation.RestController;

import java.nio.charset.StandardCharsets;
import java.util.List;
import java.util.Map;
import java.util.function.Function;
import java.util.stream.Collectors;

/**
 * Serves Connect unary procedures for the protobuf contracts this service implements.
 *
 * <p>Both wire formats are accepted: the Go clients default to binary protobuf, the browser and
 * Expo clients send proto3 JSON. Errors are always JSON, as the Connect protocol requires.
 */
@RestController
public class ConnectController {

    private static final Logger log = LoggerFactory.getLogger(ConnectController.class);

    private static final String PROTO_CONTENT_TYPE = "application/proto";

    private static final JsonFormat.Parser PARSER = JsonFormat.parser().ignoringUnknownFields();
    private static final JsonFormat.Printer PRINTER = JsonFormat.printer()
            .omittingInsignificantWhitespace()
            .alwaysPrintFieldsWithNoPresence();

    private final Map<String, ConnectService> services;

    public ConnectController(List<ConnectService> services) {
        this.services = services.stream()
                .collect(Collectors.toMap(ConnectService::serviceName, Function.identity()));
    }

    @PostMapping(path = "/{service:[A-Za-z0-9_]+\\.v[0-9]+\\.[A-Za-z0-9_]+}/{method}")
    public ResponseEntity<byte[]> handle(
            @PathVariable String service,
            @PathVariable String method,
            @RequestHeader(value = HttpHeaders.CONTENT_TYPE, required = false) String contentType,
            @RequestBody(required = false) byte[] body) {
        boolean binary = contentType != null && contentType.toLowerCase().contains("proto");

        try {
            ConnectService target = services.get(service);
            if (target == null) {
                throw ConnectException.unimplemented(service + "/" + method);
            }

            ConnectService.Procedure<?, ?> procedure = target.procedure(method);
            Message request = parse(procedure.prototype(), body, binary);
            Message response = procedure.invoke(request);

            return binary
                    ? ResponseEntity.ok()
                        .header(HttpHeaders.CONTENT_TYPE, PROTO_CONTENT_TYPE)
                        .body(response.toByteArray())
                    : ResponseEntity.ok()
                        .contentType(MediaType.APPLICATION_JSON)
                        .body(PRINTER.print(response).getBytes(StandardCharsets.UTF_8));
        } catch (ConnectException ex) {
            return error(ex.code(), ex.getMessage());
        } catch (InvalidProtocolBufferException ex) {
            return error(ConnectCode.INVALID_ARGUMENT, "the request body does not match this procedure");
        } catch (MissingFamilyException ex) {
            return error(ConnectCode.FAILED_PRECONDITION, ex.getMessage());
        } catch (ResourceNotFoundException ex) {
            return error(ConnectCode.NOT_FOUND, ex.getMessage());
        } catch (InvalidRequestException ex) {
            return error(ConnectCode.INVALID_ARGUMENT, ex.getMessage());
        } catch (RuntimeException ex) {
            log.error("connect procedure {}/{} failed", service, method, ex);
            return error(ConnectCode.INTERNAL, "internal error");
        }
    }

    private static Message parse(Message prototype, byte[] body, boolean binary)
            throws InvalidProtocolBufferException {
        Message.Builder builder = prototype.newBuilderForType();
        if (body == null || body.length == 0) {
            return builder.build();
        }
        if (binary) {
            return builder.mergeFrom(body).build();
        }
        PARSER.merge(new String(body, StandardCharsets.UTF_8), builder);
        return builder.build();
    }

    private static ResponseEntity<byte[]> error(ConnectCode code, String message) {
        String escaped = message == null ? "" : message.replace("\\", "\\\\").replace("\"", "\\\"");
        String payload = "{\"code\":\"" + code.wireName() + "\",\"message\":\"" + escaped + "\"}";
        return ResponseEntity.status(code.status())
                .contentType(MediaType.APPLICATION_JSON)
                .body(payload.getBytes(StandardCharsets.UTF_8));
    }
}
