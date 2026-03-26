Feature: First Time User Experience
  As an administrator
  I want to set up the system
  So users can start using search

  Scenario: Complete FTUE setup
    Given the API is running
    When I create an admin with email "admin@test.com" and password "password123"
    Then the response status should be 201

    When I login with email "admin@test.com" and password "password123"
    Then the response status should be 200
    And I should receive a token

    When I connect Vespa
    Then the response status should be 200

    # Wait for Vespa container to be fully ready after schema deployment
    When Vespa is fully ready
    Then the system should be healthy
