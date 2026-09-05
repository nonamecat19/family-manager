package com.example.notesjava.common.persistence;

import org.springframework.context.annotation.Configuration;
import org.springframework.data.jpa.repository.config.EnableJpaAuditing;

/**
 * Split out of the application class so slice tests can import auditing without booting
 * the whole context.
 */
@Configuration
@EnableJpaAuditing
public class JpaAuditingConfig {
}
