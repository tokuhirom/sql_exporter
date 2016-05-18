# sql_exporter

Prometheus exporter, runs query and export results.

## Usage

    ./sql_exporter -config=/path/to/config.yml

        -config=path/to/config.yml
        -log.level=(debug|info)

## Configuration

You need to write configuration file in YAML format.

Current implementation supports mysql and sqlite3.
(This exporter uses golang's `database/sql` to access database.
If you need to access other database systems, you need to use `import` statement in `sql_exporter.go`.
Patches welcome.
See also https://github.com/golang/go/wiki/SQLDrivers and https://golang.org/pkg/database/sql/)

    ---
    driver_name: mysql
    data_source_name: root:password@/myapp
    queries:
      - sql: SELECT count(id) FROM user
        name: users
        help: number of users
      - sql: SELECT count(*) cnt, blog_id FROM entry GROUP BY blog_id ORDER BY cnt desc LIMIT 10
        name: entries
        help: number of entries(in top blogs)

## Sample output

    # HELP sql_users_total number of farms
    # TYPE sql_users_total counter
    sql_users_total 10

## Labeling

You can get labels by query like following:

      - sql: SELECT count(*) cnt, blog_id FROM entry GROUP BY blog_id ORDER BY cnt desc LIMIT 10
        name: entries
        help: number of entries(in top blogs)

1st column must contain numeric value. It's a value of counter.

sql\_exporter uses other columns as labels.

## LICENSE

    The MIT License (MIT)
    Copyright (C) 2016 Tokuhiro Matsuno, http://64p.org/ <tokuhirom@gmail.com>

    Permission is hereby granted, free of charge, to any person obtaining a copy
    of this software and associated documentation files (the “Software”), to deal
    in the Software without restriction, including without limitation the rights
    to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
    copies of the Software, and to permit persons to whom the Software is
    furnished to do so, subject to the following conditions:

    The above copyright notice and this permission notice shall be included in
    all copies or substantial portions of the Software.

    THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
    IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
    FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
    AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
    LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
    OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
    THE SOFTWARE.

