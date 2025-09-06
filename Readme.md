# PeFi

### TODO

- A 100% transfer from a non fixed distribution will still 0 it out.
- Grouped transfer-templates for easier management.
    - no grouping within the prediction, just for the management in the UI.
    - this will allow setting changes for salary from certain dates etc.
    - should we allow grouping transfer templates to see them as a group of the same transfer template. Ie. salary changes between these dates and then goes to this again.
- Add categories to transfer-templates
- Model startup shares
    - Model startup shares valuation based on company valuation and shares owned by the user
    - Model startup shares prediction based on future funding rounds and exit events
- Simulation Scenario combine different transaction templates into a sceanrio and see differences

## Tasks

### Core

- Tests for predictions

## Details

Asset class - (annualReturn, annualVolatility)

### Suggested configurations (nominal returns, annualized)
#### Pension funds

AP7 Såfa (equity-heavy, with leverage if you’re <55)

Fixed: (annualReturn = 0.09, annualVolatility = 0.23)

Normal: annualReturn ~ N(0.09, 0.02), annualVolatility ~ N(0.23, 0.03)

Traditionell försäkring (with guarantees / smoothing)

Fixed: (annualReturn = 0.05, annualVolatility = 0.04)

Normal: annualReturn ~ N(0.05, 0.01), annualVolatility ~ N(0.04, 0.01)

#### Investment account (broad equity funds)

Swedish equity fund (OMXS30 / Sweden IMI)

Fixed: (annualReturn = 0.08, annualVolatility = 0.20)

Normal: annualReturn ~ N(0.08, 0.02), annualVolatility ~ N(0.20, 0.03)

#### Global index fund (MSCI World/ACWI in SEK)

Fixed: (annualReturn = 0.07, annualVolatility = 0.16)

Normal: annualReturn ~ N(0.07, 0.015), annualVolatility ~ N(0.16, 0.02)

#### Stockholm apartment (bostadsrätt)

Fixed: (annualReturn = 0.07, annualVolatility = 0.08)

Normal: annualReturn ~ N(0.07, 0.015), annualVolatility ~ N(0.08, 0.02)
