package com.example.notesjava.common.web;

import org.springframework.http.MediaType;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

/**
 * Every other service in this repo answers GET /healthz, and deploy.sh probes exactly that path
 * from inside the caddy container. Spring's own probe lives at /actuator/health; this is the
 * alias that lets the deployment treat this service like the rest.
 */
@RestController
public class HealthController {

    @GetMapping(path = "/healthz", produces = MediaType.APPLICATION_JSON_VALUE)
    public ResponseEntity<String> healthz() {
        return ResponseEntity.ok("{\"status\":\"ok\"}");
    }
}
