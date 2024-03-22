.. _reference-rest-client-disable:

#################################
Désactiver un client au démarrage
#################################

.. http:put:: /api/client/(string:client_name)/disable

   Désactive le client demandé. Celui-ci ne sera donc pas démarré au prochain
   lancement de la *gateway*, et restera donc inactif.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 202: Le client a été activé avec succès.
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le client demandé n'existe pas


   **Exemple de requête**

      .. code-block:: http

         DELETE https://my_waarp_gateway.net/api/clients/sftp_client/disable HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 202 ACCEPTED
