.. _reference-rest-servers-enable:

###############################
Activer un serveur au démarrage
###############################

.. http:put:: /api/servers/(string:server_name)/enable

   Active le serveur demandé. Celui-ci sera donc démarré au prochain lancement
   de la *gateway*.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 202: Le serveur a été activé avec succès.
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le serveur demandé n'existe pas


   **Exemple de requête**

   .. code-block:: http

      DELETE https://my_waarp_gateway.net/api/servers/sftp_server/enable HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 202 ACCEPTED
