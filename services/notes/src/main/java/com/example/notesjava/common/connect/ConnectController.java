package com.example.notesjava.common.connect;

import com.google.protobuf.Message;
import com.google.protobuf.util.JsonFormat;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.http.MediaType;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RestController;

import java.util.List;
import java.util.Map;
import java.util.function.Function;
import java.util.stream.Collectors;

@RestController
public class ConnectController {

    private static final Logger log = LoggerFactory.getLogger(ConnectController.class);

    private static final JsonFormat.Parser PARSER = JsonFormat.parser().ignoringUnknownFields();
    private static final JsonFormat.Printer PRINTER = JsonFormat.printer()
            .omittingInsignificantWhitespace().alwaysPrintFieldsWithNoPresence();

    private final Map<String, ConnectService> services;

    public ConnectController(List<ConnectService> services) {
        this.services = services.stream()
                .collect(Collectors.toMap(ConnectService::serviceName, Function.identity()));
    }

    @PostMapping(
            path = "/{service:[A-Za-z0-9_]+\\.v[0-9]+\\.[A-Za-z0-9_]+}/{method}",
            consumes = MediaType.APPLICATION_JSON_VALUE,
            produces = MediaType.APPLICATION_JSON_VALUE)
    public ResponseEntity<String> handle(
            @PathVariable String service,
            @PathVariable String method,
            @RequestBody(required = false) String body) {
        try {
            ConnectService target = services.get(service);
            if (target == null) {
                throw ConnectException.unimplemented(service + "/" + method);
            }

            ConnectService.Procedure<?, ?> procedure = target.procedure(method);
            Message.Builder builder = procedure.prototype().newBuilderForType();
            PARSER.merge(body == null || body.isBlank() ? "{}" : body, builder);

            Message response = procedure.invoke(builder.build());
            return ResponseEntity.ok(PRINTER.print(response));
        } catch (ConnectException ex) {
            return error(ex.code(), ex.getMessage());
        } catch (com.google.protobuf.InvalidProtocolBufferException ex) {
            return error(ConnectCode.INVALID_ARGUMENT, "the request body is not valid JSON for this procedure");
        } catch (RuntimeException ex) {
            log.error("connect procedure {}/{} failed", service, method, ex);
            return error(ConnectCode.INTERNAL, "internal error");
        }
    }

    private static ResponseEntity<String> error(ConnectCode code, String message) {
        String escaped = message == null ? "" : message.replace("\\", "\\\\").replace("\"", "\\\"");
        return ResponseEntity.status(code.status())
                .contentType(MediaType.APPLICATION_JSON)
                .body("{\"code\":\"" + code.wireName() + "\",\"message\":\"" + escaped + "\"}");
    }
}
