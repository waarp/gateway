Modifier les traitements d'une règle
====================================

.. http:put:: /api/rules/(int:rule_id)/tasks

   Remplace les chaînes de traitement de la règle numéro ``rule_id`` par celles
   renseignées dans la requête.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqjson array preTasks: Liste des traitements pré-transfert
   :reqjson array postTasks: Liste des traitements post-transfert
   :reqjson array errorTasks: Liste des traitements en cas d'erreur de transfert
   :reqjsonarr string type: Le type du traitement à effectuer
   :reqjsonarr string args: Les arguments du traitement sous forme d'un objet JSON

   **Exemple de requête**

       .. code-block:: http

          PUT https://my_waarp_gateway.net/api/rules/1/tasks HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
          Content-Type: application/json
          Content-Length: 185

          {
            "preTasks": [
              {
                "type": "COPY",
                "args": "{\"dst\": \"copy/destination\"}"
              }
            ],
            "postTasks": [
              {
                "type": "DELETE",
                "args": "{}"
              }
            ],
            "errorTasks": [
              {
                "type": "EXEC",
                "args": "{\"target\": \"program\"}"
              }
            ]
          }

   **Réponse**

   :statuscode 201: Les traitements de la règle ont été modifiés avec succès
   :statuscode 400: Un ou plusieurs des paramètres des traitements sont invalides
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: La règle demandée n'existe pas

   :resheader Location: Le chemin d'accès des traitements

   **Exemple de réponse**

       .. code-block:: http

          HTTP/1.1 201 CREATED
          Location: https://my_waarp_gateway.net/api/rules/1/tasks
