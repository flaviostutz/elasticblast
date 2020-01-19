# elasticblast
Elasticsearch to Blast (Bleve based) server bridge. Focused on the Document API section.

To view a demo of Netflix Conductor running over Elasticblast (Conductor "thinks" it is using ES, but Bleve is doing the job), go to https://youtu.be/IjJQ0AEoyLo

## Usage

* Create docker-compose.yml

```yml
version: '3.5'
services:
  elasticblast:
    image: flaviostutz/elasticblast
    restart: always
    ports:
      - 8200:8200
    environment:
      - LOG_LEVEL=info
      - BLAST_URL=http://blast:6000

  blast:
    image: flaviostutz/blast-indexer
    ports:
      - 6000:6000
```

* Run 

```shell
docker-compose up -d

#test cluster health status
curl --location --request GET 'localhost:8200/_cluster/health?timeout=30s&wait_for_status=green'

#test document creation
curl --location --request PUT 'localhost:8200/testi/testm/abcid' \
--header 'Content-Type: application/json' \
--data-raw '{"workflowType":"kitchensink","version":1,"workflowId":"193f4a0f-00e0-4396-9d20-3d13e28ae7b3","startTime":"2019-12-27T12:10:16.601Z","status":"RUNNING","input":"{task2Name=task_5}","output":"{}","executionTime":0,"inputSize":18,"outputSize":2}'

#test document search
curl --location --request POST 'localhost:8200/testi/testm/_search?typed_keys=true&ignore_unavailable=false&expand_wildcards=open&allow_no_indices=true&search_type=query_then_fetch&batched_reduce_size=512' \
--header 'Content-Type: application/json' \
--data-raw '{
    "from": 0,
    "size": 100,
    "query": {
        "bool": {
            "must": [
                {
                    "query_string": {
                        "query": "*",
                        "fields": []
                    }
                }
            ]
        }
    },
    "sort": [
        {
            "startTime": {
                "order": "desc"
            }
        }
    ]
}'
```

## Current ES search query support

* from
* size
* sort
* bool.must.search_query.query - only one term
* bool.must.search_query.fields
* bool.must.terms

Supported ES query example:

```json
{
    "from": 0,
    "size": 100,
    "query": {
        "bool": {
            "must": [
                {
                    "query_string": {
                        "query": "\"task2Name\"",
                        "fields": ["startTime"]
                    }
                },
		        {
				"bool": {
		            "must": [
		              {
		                "bool": {
		                  "must": [
		                    {
		                      "terms": {
		                        "workflowType": [
		                          "kitchensink"
		                        ]
		                      }
		                    },
		                    {
		                      "terms": {
		                        "status": [
		                          "RUNNING"
		                        ]
		                      }
		                    }
		                  ]
		                }
		              }
		            ]
		          }
		        }
            ]
        }
    },
    "sort": [
        {
            "startTime": {
                "order": "desc"
            }
        }
    ]
}
```

## Warning

As a complete adapter would be too expensive to implement, so some features from ES are ignored. We will evolve this adapter as needed. Currently basic Document creation, update and search is implemented.

Until now, we use it for replacing Elasticsearch in Netflix Conductor server (we've done a lot of testing there!).
