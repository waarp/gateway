.. _reference-rest-servers-authorize:

##########################################
Autoriser un serveur à utiliser une règle
##########################################

.. http:put:: /api/servers/(string:server_name)/authorize/(string:rule)

   Authorise le serveur demandé à utiliser la règle donnée. Donner une permission
   à un serveur donne automatiquement cette permission à tous les comptes rattachés
   à ce serveur.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 200: La permission a été donnée avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le serveur ou la règle demandés n'existent pas


   |

   **Exemple de requête**

      .. code-block:: http

         DELETE https://my_waarp_gateway.net/api/servers/sftp_server/authorize/rule_1 HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 200 OK
