Autoriser un compte à utiliser une règle
=========================================

.. http:put:: /api/servers/(string:server_name)/accounts/(string:login)/authorize/(string:rule)

   Authorise le compte demandé à utiliser la règle donnée. Cette permission persistera,
   même si le serveur parent se fait retirer le droit d'utilisation de la règle.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 200: La permission a été donnée avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le serveur, le compte ou la règle demandés n'existent pas

   |

   **Exemple de requête**

      .. code-block:: http

         DELETE https://my_waarp_gateway.net/api/servers/sftp_server/accounts/toto/authorize/rule_1 HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 200 OK