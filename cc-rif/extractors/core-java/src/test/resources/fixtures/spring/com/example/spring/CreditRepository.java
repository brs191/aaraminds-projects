package com.example.spring;

import org.springframework.stereotype.Repository;

@Repository
public interface CreditRepository {
    void save(Object obj);
}
