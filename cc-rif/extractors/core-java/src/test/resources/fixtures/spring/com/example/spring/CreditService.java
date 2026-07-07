package com.example.spring;

import com.example.spring.dto.CreditRequest;
import com.example.spring.dto.CreditResponse;
import org.springframework.stereotype.Service;
import lombok.extern.slf4j.Slf4j;

@Slf4j
@Service
public class CreditService {
    private final CreditRepository repository;

    public CreditService(CreditRepository repository) {
        this.repository = repository;
    }

    public CreditResponse process(CreditRequest request) {
        return validate(request);
    }

    private CreditResponse validate(CreditRequest request) {
        return new CreditResponse();
    }
}
