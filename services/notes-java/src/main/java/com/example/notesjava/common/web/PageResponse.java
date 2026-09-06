package com.example.notesjava.common.web;

import org.springframework.data.domain.Page;

import java.util.List;

/**
 * Wire shape for a page. Spring's own {@code Page} serialises its internals, which are not a
 * contract; this is.
 */
public record PageResponse<T>(
        List<T> content,
        int page,
        int size,
        long totalElements,
        int totalPages
) {
    public static <T> PageResponse<T> of(Page<T> page) {
        return new PageResponse<>(
                page.getContent(),
                page.getNumber(),
                page.getSize(),
                page.getTotalElements(),
                page.getTotalPages()
        );
    }
}
