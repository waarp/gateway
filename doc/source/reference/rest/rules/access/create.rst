Accorder un accès à une règle
=============================

.. http:post:: /api/rules/(int:rule_id)/access

   Ajoute un accès pour l'entitée renseignée en JSON à la règle portant le numéro
   ``rule_id``.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqjson string objectType: Le type d'entitée à laquelle l'accès est accordé
                               (valeurs possibles: *local_agents*, *remote_agents*,
                               *local_accounts* et *remote_accounts*)
   :reqjson number objectID: L'identifiant de l'entitée

   **Exemple de requête**

       .. code-block:: http

          POST https://my_waarp_gateway.net/api/rules/1/access HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
          Content-Type: application/json
          Content-Length: 78

          {
            "objectType": "local_agents",
            "objectID": 1,
          }

   **Réponse**

   :statuscode 201: L'accès à la règle a été créé avec succès
   :statuscode 400: Un ou plusieurs des paramètres de l'accès sont invalides
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: La règle demandée n'existe pas

   :resheader Location: Le chemin d'accès du nouvel accès créé

   **Exemple de réponse**

       .. code-block:: http

          HTTP/1.1 201 CREATED
          Location: https://my_waarp_gateway.net/api/rules/1/access
