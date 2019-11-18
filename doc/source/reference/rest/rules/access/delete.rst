Révoquer un accès à une règle
=============================

.. http:delete:: /api/rules/(int:rule_id)/access

   Révoque l'accès de l'entitée renseignée en JSON à la règle portant le numéro
   ``rule_id``.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqjson string objectType: Le type d'entitée à laquelle l'accès est accordé
                               (valeurs possibles: *local_agents*, *remote_agents*,
                               *local_accounts* et *remote_accounts*)
   :reqjson number objectID: L'identifiant de l'entitée

   **Exemple de requête**

       .. code-block:: http

          DELETE https://my_waarp_gateway.net/api/rules/1/access HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
          Content-Type: application/json
          Content-Length: 78

          {
            "objectType": "local_agents",
            "objectID": 1,
          }


   **Réponse**

   :statuscode 204: L'accès à la règle a été supprimé avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: La règle demandée n'existe pas

   **Exemple de réponse**

       .. code-block:: http

          HTTP/1.1 204 NO CONTENT