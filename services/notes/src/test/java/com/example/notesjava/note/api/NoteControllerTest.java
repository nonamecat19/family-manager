package com.example.notesjava.note.api;

import com.example.notesjava.common.error.ResourceNotFoundException;
import com.example.notesjava.note.api.dto.CreateNoteRequest;
import com.example.notesjava.note.api.dto.NoteResponse;
import com.example.notesjava.note.domain.Note;
import com.example.notesjava.note.domain.NotePriority;
import com.example.notesjava.note.domain.NoteStatus;
import com.example.notesjava.note.service.NoteService;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import com.example.notesjava.common.security.SecurityConfig;
import org.springframework.boot.webmvc.test.autoconfigure.WebMvcTest;
import org.springframework.context.annotation.Import;
import org.springframework.http.MediaType;
import org.springframework.test.context.bean.override.mockito.MockitoBean;
import org.springframework.test.web.servlet.MockMvc;
import tools.jackson.databind.ObjectMapper;

import java.time.Instant;

import static org.hamcrest.Matchers.endsWith;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.when;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.delete;
import static org.springframework.security.test.web.servlet.request.SecurityMockMvcRequestPostProcessors.jwt;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.header;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

@WebMvcTest(NoteController.class)
@Import(SecurityConfig.class)
class NoteControllerTest {

    @Autowired
    private MockMvc mockMvc;

    @Autowired
    private ObjectMapper objectMapper;

    @MockitoBean
    private NoteService noteService;

    private static final org.springframework.test.web.servlet.request.RequestPostProcessor CALLER =
            jwt().jwt(token -> token
                    .subject("11111111-1111-1111-1111-111111111111")
                    .claim("family_id", "22222222-2222-2222-2222-222222222222"));

    @Test
    void anUnauthenticatedRequestIsA401() throws Exception {
        mockMvc.perform(get("/api/notes"))
                .andExpect(status().isUnauthorized())
                .andExpect(jsonPath("$.title").value("Unauthenticated"));
    }

    @Test
    void rejectsABlankTitleWithAProblemDetail() throws Exception {
        mockMvc.perform(post("/api/notes").with(CALLER).contentType(MediaType.APPLICATION_JSON)
                        .content(json(new CreateNoteRequest("   ", "content", null, null, null, null))))
                .andExpect(status().isBadRequest())
                .andExpect(jsonPath("$.title").value("Validation failed"))
                .andExpect(jsonPath("$.errors.title").exists());
    }

    @Test
    void rejectsATitleOverTheLengthLimit() throws Exception {
        mockMvc.perform(post("/api/notes").with(CALLER).contentType(MediaType.APPLICATION_JSON)
                        .content(json(new CreateNoteRequest("x".repeat(256), null, null, null, null, null))))
                .andExpect(status().isBadRequest())
                .andExpect(jsonPath("$.errors.title").exists());
    }

    @Test
    void createReturns201WithALocationHeader() throws Exception {
        when(noteService.create(any())).thenReturn(response(1L, "title"));

        mockMvc.perform(post("/api/notes").with(CALLER).contentType(MediaType.APPLICATION_JSON)
                        .content(json(new CreateNoteRequest("title", "content", null, null, null, null))))
                .andExpect(status().isCreated())
                .andExpect(header().string("Location", endsWith("/api/notes/1")))
                .andExpect(jsonPath("$.title").value("title"));
    }

    @Test
    void missingNoteBecomesA404ProblemDetail() throws Exception {
        when(noteService.getById(404L)).thenThrow(new ResourceNotFoundException("Note", 404L));

        mockMvc.perform(get("/api/notes/404").with(CALLER))
                .andExpect(status().isNotFound())
                .andExpect(jsonPath("$.title").value("Resource not found"))
                .andExpect(jsonPath("$.detail").value("Note 404 not found"));
    }

    @Test
    void deleteReturns204() throws Exception {
        mockMvc.perform(delete("/api/notes/1").with(CALLER)).andExpect(status().isNoContent());
    }

    @Test
    void anUnsupportedMethodStaysA405() throws Exception {
        mockMvc.perform(org.springframework.test.web.servlet.request.MockMvcRequestBuilders
                        .patch("/api/notes/1").with(CALLER))
                .andExpect(status().isMethodNotAllowed());
    }

    @Test
    void anUnparseableBodyStaysA400() throws Exception {
        mockMvc.perform(post("/api/notes").with(CALLER).contentType(MediaType.APPLICATION_JSON).content("{"))
                .andExpect(status().isBadRequest());
    }

    private String json(Object value) {
        return objectMapper.writeValueAsString(value);
    }

    private static NoteResponse response(Long id, String title) {
        Instant now = Instant.now();
        return new NoteResponse(id, title, "content", NoteStatus.ACTIVE, NotePriority.NORMAL,
                null, null, 0L, now, now);
    }
}
