.. _reference-rest-client-enable:

##############################
Activer un client au démarrage
##############################

.. http:put:: /api/client/(string:client_name)/enable

   Active le client demandé. Celui-ci sera donc démarré au prochain lancement
   de la *gateway*.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 202: Le client a été activé avec succès.
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le client demandé n'existe pas


   **Exemple de requête**

      .. code-block:: http

         DELETE https://my_waarp_gateway.net/api/clients/sftp_client/enable HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 202 ACCEPTED
