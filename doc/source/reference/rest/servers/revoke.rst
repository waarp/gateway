Interdire à un serveur d'utiliser une règle
===========================================

.. http:put:: /api/servers/(string:server_name)/revoke/(string:rule)

   Retire au serveur demandé la permission d'utiliser la règle donnée. Retire
   également la permission à tous les comptes rattachés à ce serveur n'ayant pas
   été explicitement autorisés.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 200: La permission a été donnée avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le serveur ou la règle demandés n'existent pas


   .. admonition:: Exemple de requête

      .. code-block:: http

         DELETE https://my_waarp_gateway.net/api/servers/sftp_server/authorize/rule_1 HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   .. admonition:: Exemple de réponse

      .. code-block:: http

         HTTP/1.1 200 OK