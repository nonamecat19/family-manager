package com.example.notesjava.common.connect;

import com.example.notesjava.common.security.SecurityConfig;
import com.nnc.familymanager.notes.v1.ListNotebooksRequest;
import com.nnc.familymanager.notes.v1.ListNotebooksResponse;
import com.nnc.familymanager.notes.v1.Notebook;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.TestConfiguration;
import org.springframework.boot.webmvc.test.autoconfigure.WebMvcTest;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Import;
import org.springframework.http.MediaType;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.test.web.servlet.request.RequestPostProcessor;

import static org.springframework.security.test.web.servlet.request.SecurityMockMvcRequestPostProcessors.jwt;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

@WebMvcTest(ConnectController.class)
@Import({SecurityConfig.class, ConnectControllerTest.Fixtures.class})
class ConnectControllerTest {

    private static final String LIST_NOTEBOOKS = "/notes.v1.NotesService/ListNotebooks";

    private static final RequestPostProcessor CALLER = jwt().jwt(token -> token
            .subject("11111111-1111-1111-1111-111111111111")
            .claim("family_id", "22222222-2222-2222-2222-222222222222"));

    @Autowired
    private MockMvc mockMvc;

    @Autowired
    private FakeNotesService notes;

    @BeforeEach
    void reset() {
        notes.answer(request -> ListNotebooksResponse.newBuilder()
                .addNotebooks(Notebook.newBuilder().setId("7").setName("Kitchen").setNoteCount(2))
                .build());
    }

    @Test
    void anUnauthenticatedCallIs401() throws Exception {
        mockMvc.perform(post(LIST_NOTEBOOKS).contentType(MediaType.APPLICATION_JSON).content("{}"))
                .andExpect(status().isUnauthorized());
    }

    @Test
    void aProcedureAnswersWithProtoJson() throws Exception {
        mockMvc.perform(post(LIST_NOTEBOOKS).with(CALLER)
                        .contentType(MediaType.APPLICATION_JSON).content("{}"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.notebooks[0].id").value("7"))
                .andExpect(jsonPath("$.notebooks[0].name").value("Kitchen"))
                .andExpect(jsonPath("$.notebooks[0].noteCount").value(2));
    }

    @Test
    void anUnknownProcedureIsUnimplemented() throws Exception {
        mockMvc.perform(post("/notes.v1.NotesService/Nope").with(CALLER)
                        .contentType(MediaType.APPLICATION_JSON).content("{}"))
                .andExpect(status().isNotImplemented())
                .andExpect(jsonPath("$.code").value("unimplemented"));
    }

    @Test
    void anUnknownServiceIsUnimplemented() throws Exception {
        mockMvc.perform(post("/shopping.v1.ShoppingService/ListItems").with(CALLER)
                        .contentType(MediaType.APPLICATION_JSON).content("{}"))
                .andExpect(status().isNotImplemented())
                .andExpect(jsonPath("$.code").value("unimplemented"));
    }

    @Test
    void aDomainErrorKeepsItsConnectCode() throws Exception {
        notes.answer(request -> {
            throw ConnectException.notFound("that notebook does not exist");
        });

        mockMvc.perform(post(LIST_NOTEBOOKS).with(CALLER)
                        .contentType(MediaType.APPLICATION_JSON).content("{}"))
                .andExpect(status().isNotFound())
                .andExpect(jsonPath("$.code").value("not_found"))
                .andExpect(jsonPath("$.message").value("that notebook does not exist"));
    }

    @Test
    void malformedJsonIsAnInvalidArgumentNotACrash() throws Exception {
        mockMvc.perform(post(LIST_NOTEBOOKS).with(CALLER)
                        .contentType(MediaType.APPLICATION_JSON).content("{\"notebooks\": "))
                .andExpect(status().isBadRequest())
                .andExpect(jsonPath("$.code").value("invalid_argument"));
    }

    @Test
    void unknownFieldsAreIgnoredRatherThanRejected() throws Exception {
        mockMvc.perform(post(LIST_NOTEBOOKS).with(CALLER)
                        .contentType(MediaType.APPLICATION_JSON).content("{\"somethingNewer\":true}"))
                .andExpect(status().isOk());
    }

    @Test
    void anUnexpectedFailureStaysOpaque() throws Exception {
        notes.answer(request -> {
            throw new IllegalStateException("connection reset by peer at 10.0.0.5");
        });

        mockMvc.perform(post(LIST_NOTEBOOKS).with(CALLER)
                        .contentType(MediaType.APPLICATION_JSON).content("{}"))
                .andExpect(status().isInternalServerError())
                .andExpect(jsonPath("$.code").value("internal"))
                .andExpect(jsonPath("$.message").value("internal error"));
    }

    static class FakeNotesService extends ConnectService {

        private volatile java.util.function.Function<ListNotebooksRequest, ListNotebooksResponse> handler;

        FakeNotesService() {
            register("ListNotebooks", ListNotebooksRequest.getDefaultInstance(),
                    (ListNotebooksRequest request) -> handler.apply(request));
        }

        void answer(java.util.function.Function<ListNotebooksRequest, ListNotebooksResponse> next) {
            this.handler = next;
        }

        @Override
        public String serviceName() {
            return "notes.v1.NotesService";
        }
    }

    @TestConfiguration
    static class Fixtures {

        @Bean
        FakeNotesService fakeNotesService() {
            return new FakeNotesService();
        }
    }
}
