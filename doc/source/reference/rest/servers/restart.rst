Redémarrer un serveur
=====================

.. http:delete:: /api/servers/(string:server_name)/restart

   Arrête et relance le serveur demandé.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 204: Le serveur a été redémarré avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le serveur demandé n'existe pas


   **Exemple de requête**

      .. code-block:: http

         DELETE https://my_waarp_gateway.net/api/servers/sftp_server/restart HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 202 ACCEPTED
