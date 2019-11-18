Lister les accès à une règle
============================

.. http:get:: /api/rules/(int:rule_id)/access

   Renvoie une liste de tous les accès accordés à la règle portant le numéro
   ``rule_id``.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   **Exemple de requête**

       .. code-block:: http

          GET https://my_waarp_gateway.net/api/rules/1/access HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==


   **Réponse**

   :statuscode 200: La liste a été renvoyée avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: La règle demandée n'existe pas

   :resjson array permissions: La liste des accès demandés
   :resjsonarr string objectType: Le type d'entitée à laquelle l'accès est accordé
                               (valeurs possibles: *local_agents*, *remote_agents*,
                               *local_accounts* et *remote_accounts*)
   :resjsonarr number objectID: L'identifiant de l'entitée

   **Exemple de réponse**

       .. code-block:: http

          HTTP/1.1 200 OK
          Content-Type: application/json
          Content-Length: 134

          {
            "permissions": [{
              "objectType": "local_agents",
              "objectID": 1,
            },{
              "objectType": "remote_accounts",
              "objectID": 2,
            }]
          }