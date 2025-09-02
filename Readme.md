# PeFi

### TODO

- A 100% transfer from a non fixed distribution will still 0 it out.
- Grouped transfer-templates for easier management.
    - no grouping within the prediction, just for the management in the UI.
    - this will allow setting changes for salary from certain dates etc.
- Add categories to transfer-templates
- Add transfer run for a specific date for doing the money depositing split for that month
- Model startup shares
    - Model startup shares valuation based on company valuation and shares owned by the user
    - Model startup shares prediction based on future funding rounds and exit events
- Simulation Scenario combine different transaction templates into a sceanrio and see differences
- Should we allow grouping transfer templates to see them as a group of the same transfer template. Ie. salary changes between these dates and then goes to this again.

## Tasks

### Core

- Tests for predictions

### Pages

#### Table

- color the cells based on if balance increased or decreased

## Details

Asset class - (annualReturn, annualVolatility)

- Savings (0-2%, 0%)
- Funds (4%-8%, 6%-12%)
- Stocks (7%-10%, 15%-20%)
- Startup Stocks (20%-100%, 50%-200%)
- Private Company Stocks (5%-15%, 20%-40%)
- Real Estate (6%-10%, 10%-20%)
