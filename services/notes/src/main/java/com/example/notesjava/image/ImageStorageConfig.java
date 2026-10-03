package com.example.notesjava.image;

import org.springframework.boot.autoconfigure.condition.ConditionalOnExpression;
import org.springframework.boot.context.properties.EnableConfigurationProperties;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Configuration
@EnableConfigurationProperties(StorageProperties.class)
public class ImageStorageConfig {

    @Bean
    @ConditionalOnExpression("'${notes.storage.endpoint:}'.trim() != ''")
    ImageStore imageStore(StorageProperties properties) {
        return new S3ImageStore(properties);
    }
}
