Lister les traitements d'une règle
==================================

.. http:get:: /api/rules/(int:rule_id)/tasks

   Renvoie une liste de tous les traitements de la règle portant le numéro
   ``rule_id``.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   **Exemple de requête**

       .. code-block:: http

          GET https://my_waarp_gateway.net/api/rules/1/tasks HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==


   **Réponse**

   :statuscode 200: La liste a été renvoyée avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: La règle demandée n'existe pas

   :resjson array preTasks: Liste des traitements pré-transfert
   :resjson array postTasks: Liste des traitements post-transfert
   :resjson array errorTasks: Liste des traitements en cas d'erreur de transfert
   :resjsonarr string type: Le type du traitement à effectuer
   :resjsonarr string args: Les arguments du traitement sous forme d'un objet JSON

   **Exemple de réponse**

       .. code-block:: http

          HTTP/1.1 200 OK
          Content-Type: application/json
          Content-Length: 185

          {
            "preTasks": {
              "type": "COPY",
              "args": "{\"dst\": \"copy/destination\"}"
            },
            "postTasks":
              "type": "DELETE",
              "args": "{}"
            },
            "errorTasks": {
              "type": "EXEC",
              "args": "{\"target\": \"program\"}"
            }
          }